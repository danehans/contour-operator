// Copyright Project Contour Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"
	objcontour "github.com/projectcontour/contour-operator/internal/objects/contour"
	objgc "github.com/projectcontour/contour-operator/internal/objects/gatewayclass"
	operatorconfig "github.com/projectcontour/contour-operator/internal/operator/config"
	retryable "github.com/projectcontour/contour-operator/internal/retryableerror"
	"github.com/projectcontour/contour-operator/pkg/slice"

	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1alpha1 "sigs.k8s.io/gateway-api/apis/v1alpha1"
)

const gatewayClassNamespacedParamRef = "Namespace"

// Contour returns true if contour is valid.
func Contour(ctx context.Context, cli client.Client, contour *operatorv1alpha1.Contour) error {
	// TODO [danehans]: Remove when https://github.com/projectcontour/contour-operator/issues/18 is fixed.
	exist, err := objcontour.OtherContoursExistInSpecNs(ctx, cli, contour)
	if err != nil {
		return fmt.Errorf("failed to verify if other contours exist in namespace %s: %w",
			contour.Spec.Namespace.Name, err)
	}
	if exist {
		return fmt.Errorf("other contours exist in namespace %s", contour.Spec.Namespace.Name)
	}
	return containerPorts(contour)
}

// containerPorts validates container ports of contour, returning an
// error if the container ports do not meet the API specification.
func containerPorts(contour *operatorv1alpha1.Contour) error {
	var numsFound []int32
	var namesFound []string
	httpFound := false
	httpsFound := false
	for _, port := range contour.Spec.NetworkPublishing.Envoy.ContainerPorts {
		if len(numsFound) > 0 && slice.ContainsInt32(numsFound, port.PortNumber) {
			return fmt.Errorf("duplicate container port number %q", port.PortNumber)
		}
		numsFound = append(numsFound, port.PortNumber)
		if len(namesFound) > 0 && slice.ContainsString(namesFound, port.Name) {
			return fmt.Errorf("duplicate container port name %q", port.Name)
		}
		namesFound = append(namesFound, port.Name)
		switch {
		case port.Name == "http":
			httpFound = true
		case port.Name == "https":
			httpsFound = true
		}
	}
	if httpFound && httpsFound {
		return nil
	}
	return fmt.Errorf("http and https container ports are unspecified")
}

// GatewayClass returns nil if gc is a valid GatewayClass. If gc references a
// Contour, it's returned.
func GatewayClass(ctx context.Context, cli client.Client, gc *gatewayv1alpha1.GatewayClass) (*operatorv1alpha1.Contour, error) {
	cntr, err := parameterRef(ctx, cli, gc)
	if err != nil {
		return nil, err
	}
	return cntr, nil
}

// parameterRef returns nil if parametersRef of gc is valid. If gc references a
// Contour, it's returned.
func parameterRef(ctx context.Context, cli client.Client, gc *gatewayv1alpha1.GatewayClass) (*operatorv1alpha1.Contour, error) {
	if gc.Spec.ParametersRef == nil {
		return nil, nil
	}
	if gc.Spec.ParametersRef.Scope != gatewayClassNamespacedParamRef {
		return nil, fmt.Errorf("invalid parametersRef for gateway class %s, scope %s is required",
			gc.Name, gatewayClassNamespacedParamRef)
	}
	group := gc.Spec.ParametersRef.Group
	if group != operatorv1alpha1.GatewayClassParamsRefGroup {
		return nil, fmt.Errorf("invalid group %q", group)
	}
	kind := gc.Spec.ParametersRef.Kind
	if kind != operatorv1alpha1.GatewayClassParamsRefKind {
		return nil, fmt.Errorf("invalid kind %q", kind)
	}
	ns := gc.Spec.ParametersRef.Namespace
	defaultNs := operatorconfig.DefaultNamespace
	if ns != defaultNs {
		return nil, fmt.Errorf("invalid namespace %q; only namespace %s is supported", ns, defaultNs)
	}
	name := gc.Spec.ParametersRef.Name
	cntr, err := objcontour.CurrentContour(ctx, cli, ns, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get contour %s/%s for gatewayclass %s: %w", ns, name, gc.Name, err)
	}
	return cntr, nil
}

// Gateway returns an error if gw is an invalid Gateway.
func Gateway(ctx context.Context, cli client.Client, gw *gatewayv1alpha1.Gateway) error {
	var errs []error

	gc, err := objgc.Get(ctx, cli, gw.Spec.GatewayClassName)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to get gatewayclass for gateway %s/%s: %w",
			gw.Namespace, gw.Name, err))
	}
	admitted, err := objgc.Admitted(ctx, cli, gc.Name)
	if err != nil {
		errs = append(errs, err)
	}
	if !admitted {
		errs = append(errs, fmt.Errorf("gatewayclass %s is not admitted", gc.Name))
	}
	if err := gatewayListeners(gw); err != nil {
		errs = append(errs, fmt.Errorf("failed to validate listeners for gateway %s/%s: %w", gw.Namespace,
			gw.Name, err))
	}
	if err := gatewayAddresses(gw); err != nil {
		errs = append(errs, fmt.Errorf("failed to validate addresses for gateway %s/%s: %w", gw.Namespace,
			gw.Name, err))
	}
	if len(errs) != 0 {
		return retryable.NewMaybeRetryableAggregate(errs)
	}
	return nil
}

// gatewayListeners returns an error if the listeners of the provided gw are invalid.
// TODO [danehans]: Refactor when more than 2 listeners are supported.
func gatewayListeners(gw *gatewayv1alpha1.Gateway) error {
	listeners := gw.Spec.Listeners
	if len(listeners) != 2 {
		return fmt.Errorf("%d is an invalid number of listeners", len(gw.Spec.Listeners))
	}
	if listeners[0].Port == listeners[1].Port {
		return fmt.Errorf("invalid listeners, port %v is non-unique", listeners[0].Port)
	}
	for _, listener := range listeners {
		if listener.Protocol != gatewayv1alpha1.HTTPProtocolType && listener.Protocol != gatewayv1alpha1.HTTPSProtocolType {
			return fmt.Errorf("invalid listener protocol %s", listener.Protocol)
		}
		// TODO [danehans]: Enable TLS validation for HTTPS/TLS listeners.
		// xref: https://github.com/projectcontour/contour-operator/issues/214
		empty := gatewayv1alpha1.Hostname("")
		wildcard := gatewayv1alpha1.Hostname("*")
		if listener.Hostname != nil {
			if listener.Hostname != &empty || listener.Hostname != &wildcard {
				hostname := string(*listener.Hostname)
				// According to the Gateway spec, a listener hostname cannot be an IP address.
				if ip := validation.IsValidIP(hostname); ip == nil {
					return fmt.Errorf("invalid listener hostname %s", hostname)
				}
				if parsed := validation.IsDNS1123Subdomain(hostname); parsed != nil {
					return fmt.Errorf("invalid listener hostname %s", hostname)
				}
			}
		}
	}
	// TODO [danehans]: Validate routes of a gateway.
	// xref: https://github.com/projectcontour/contour-operator/issues/215

	return nil
}

// gatewayAddresses returns an error if any gw addresses are invalid.
// TODO [danehans]: Refactor when named addresses are supported.
func gatewayAddresses(gw *gatewayv1alpha1.Gateway) error {
	if len(gw.Spec.Addresses) > 0 {
		for _, a := range gw.Spec.Addresses {
			if a.Type != gatewayv1alpha1.IPAddressType {
				return fmt.Errorf("invalid address type %v", a.Type)
			}
			if ip := validation.IsValidIP(a.Value); ip != nil {
				return fmt.Errorf("invalid address value %s", a.Value)
			}
		}
	}
	return nil
}
