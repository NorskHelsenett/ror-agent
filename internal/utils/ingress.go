// Package utils provides utility functions for the ROR agent.
// It contains helper functions for handling Kubernetes resources
// and managing their representation in the ROR system.
package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/NorskHelsenett/ror/pkg/apicontracts"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	networkingV1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// GetIngressDetails extracts information from a Kubernetes Ingress resource and converts it to an apicontracts.Ingress.
// It gathers details about the Ingress rules, associated services, IP addresses, and paths.
// The function also evaluates the health status of the ingress based on its configuration.
//
// Parameters:
//   - ctx: Context for the operation.
//   - ingress: Pointer to a Kubernetes Ingress resource to extract information from.
//
// Returns:
//   - *apicontracts.Ingress: A pointer to the constructed Ingress object with complete details.
//   - error: An error if the ingress is invalid or if there was a problem fetching related information.
func GetIngressDetails(ctx context.Context, ingress *networkingV1.Ingress) (*apicontracts.Ingress, error) {
	var newIngress apicontracts.Ingress
	ingressNameSpace := ingress.Namespace
	ingressName := ingress.Name
	ingressClassName := ""

	var rules []apicontracts.IngressRule
	var health apicontracts.Health = 1

	if ingress.Spec.IngressClassName != nil {
		ingressClassName = *ingress.Spec.IngressClassName
	}

	k8sConfig := config.GetConfigOrDie()
	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		rlog.Error("error in config", err)
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	if ingress.Spec.Rules == nil {
		return nil, fmt.Errorf("invalid ingress - missing rules")
	}

	for ruleindex, irule := range ingress.Spec.Rules {
		rlog.Debug("rule for host", rlog.String("host", irule.Host))
		rules = append(rules, apicontracts.IngressRule{
			Hostname:    irule.Host,
			IPAddresses: nil,
			Paths:       nil,
		})

		if ingress.Status.LoadBalancer.Ingress == nil {
			rlog.Debug("Ingress has no IP-address", rlog.String("ingress", ingress.Name))
		} else {
			for _, is := range ingress.Status.LoadBalancer.Ingress {
				if is.Hostname == irule.Host {
					rules[ruleindex].IPAddresses = append(rules[ruleindex].IPAddresses, is.IP)
				}
			}
		}

		// Check if HTTP is nil before trying to access its Paths
		if irule.IngressRuleValue.HTTP == nil {
			rlog.Debug("Ingress rule has no HTTP paths defined",
				rlog.String("ingress", ingress.Name),
				rlog.String("host", irule.Host))
			continue
		}

		for _, irulepath := range irule.IngressRuleValue.HTTP.Paths {
			rlog.Debug("rule for path", rlog.String("path", irulepath.Path))
			rlog.Debug("", rlog.String("service", irulepath.Backend.Service.Name))
			service, err := GetIngressService(ctx, k8sClient, ingressNameSpace, irulepath.Backend.Service.Name)
			if err != nil {
				// Log the error and continue with an empty service, or handle as appropriate
				rlog.Error("failed to get ingress service details", err, rlog.String("service", irulepath.Backend.Service.Name))
				rules[ruleindex].Paths = append(rules[ruleindex].Paths, apicontracts.IngressPath{
					Path:    irulepath.Path,
					Service: apicontracts.Service{}, // Empty service
				})
				continue
			}
			rules[ruleindex].Paths = append(rules[ruleindex].Paths, apicontracts.IngressPath{
				Path:    irulepath.Path,
				Service: service,
			})
		}
	}

	newIngress = apicontracts.Ingress{
		UID:       string(ingress.UID),
		Health:    health,
		Name:      ingressName,
		Namespace: ingressNameSpace,
		Class:     ingressClassName,
		Rules:     rules,
	}

	return GetIngressHealth(newIngress)
}

// GetIngressHealth evaluates the health status of an Ingress resource based on predefined criteria.
// It checks for valid ingress class, presence of rules, IP addresses, paths, and service configurations.
// Health status is updated in the Ingress object itself.
//
// Parameters:
//   - thisIngress: The apicontracts.Ingress object to evaluate health for.
//
// Returns:
//   - *apicontracts.Ingress: A pointer to the same Ingress object with updated health status.
//   - error: Error if any issues occur during health evaluation.
func GetIngressHealth(thisIngress apicontracts.Ingress) (*apicontracts.Ingress, error) {

	ingressClasses := []string{"internett", "helsenett", "datacenter"}
	thisIngressClass := strings.Split(thisIngress.Class, "-")[len(strings.Split(thisIngress.Class, "-"))-1]

	if !slices.Contains(ingressClasses, thisIngressClass) {
		thisIngress.Health = 3
	}
	if len(thisIngress.Rules) < 1 {
		thisIngress.Health = 3
	} else {
		for _, rule := range thisIngress.Rules {
			if len(rule.IPAddresses) < 1 {
				thisIngress.Health = 3
			}
			if len(rule.Paths) < 1 {
				thisIngress.Health = 3
			} else {
				for _, path := range rule.Paths {
					if path.Service.Type != "NodePort" {
						thisIngress.Health = 3
					}
					// Corrected condition: check if there are no endpoints
					if len(path.Service.Endpoints) == 0 {
						thisIngress.Health = 3
					}
				}
			}
		}
	}

	return &thisIngress, nil

}

// GetIngressService retrieves detailed information about a Kubernetes Service associated with an Ingress.
// It fetches service details including type, ports, selectors, and endpoints.
//
// Parameters:
//   - ctx: Context for the operation.
//   - k8sClient: Kubernetes client to use for API calls.
//   - namespace: Namespace where the service is located.
//   - serviceName: Name of the service to retrieve information for.
//
// Returns:
//   - apicontracts.Service: A Service object containing details about the requested service.
//   - error: Error if any issues occur while retrieving service information.
func GetIngressService(ctx context.Context, k8sClient *kubernetes.Clientset, namespace string, serviceName string) (apicontracts.Service, error) {

	var service apicontracts.Service
	var endpoints []apicontracts.EndpointAddress
	var ports []apicontracts.ServicePort

	listOptions := metav1.ListOptions{}
	svcs, err := k8sClient.CoreV1().Services(namespace).List(ctx, listOptions)
	if err != nil {
		rlog.Error("could not list svcs", err, rlog.String("namespace", namespace))
		return apicontracts.Service{}, fmt.Errorf("failed to list services in namespace %s: %w", namespace, err)
	}
	for _, svc := range svcs.Items {
		if svc.Name == serviceName {
			for _, port := range svc.Spec.Ports {
				ports = append(ports, apicontracts.ServicePort{
					Name:     port.Name,
					NodePort: fmt.Sprint(port.NodePort),
					Protocol: string(port.Protocol),
				})
			}

			service = apicontracts.Service{
				Name:      serviceName,
				Type:      string(svc.Spec.Type),
				Selector:  svc.Spec.Selector["app.kubernetes.io/name"],
				Ports:     ports,
				Endpoints: nil,
			}
			rlog.Debug("service added ", rlog.String("service", serviceName))
		}
	}

	if service.Name == "" {
		service = apicontracts.Service{
			Name:      serviceName,
			Type:      "",
			Selector:  "",
			Ports:     nil,
			Endpoints: nil,
		}
		rlog.Debug("Could not find Service", rlog.String("service name", serviceName))
	}

	eps, err := k8sClient.CoreV1().Endpoints(namespace).List(ctx, listOptions)
	if err != nil {
		rlog.Error("could not list eps", err, rlog.String("namespace", namespace))
		return service, fmt.Errorf("failed to list endpoints in namespace %s for service %s: %w", namespace, serviceName, err)
	}
	for _, ep := range eps.Items {
		if ep.Name != serviceName {
			continue
		}

		if ep.Subsets == nil {
			continue
		}

		for _, epAddress := range ep.Subsets[0].Addresses {
			nodename := "None"
			if epAddress.NodeName != nil {
				nodename = *epAddress.NodeName
			}
			podname := "None"
			if epAddress.TargetRef != nil {
				podname = epAddress.TargetRef.Name
			}
			endpoints = append(endpoints, apicontracts.EndpointAddress{
				NodeName: nodename,
				PodName:  podname,
			})
			service.Endpoints = endpoints
		}

	}

	return service, nil

}
