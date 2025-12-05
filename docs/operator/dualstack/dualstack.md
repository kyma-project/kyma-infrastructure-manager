# Kyma Runtime with Dual-Stack Support

On the Google Cloud and Amazon Web Services IaaS providers, SAP BTP, Kyma runtime can support both Internet Protocol (IP) versions IPv4 and IPv6 in parallel (dual stack).

To retrieve support for both Internet protocols, create your Kyma runtime instance with dual-stack support enabled.

This feature is only available for newly created Kyma runtime instances. For existing runtimes, adding support for IPv6 afterwards is not possible, so they only use IPv4. The exception is single-stack clusters running on Amazon Web Services. For these clusters, you can enable support for public IPv6 traffic. See the hint in the [External Load Balancer: AWS](#amazon-web-services) section.

When you create a Kyma cluster with dual-stack support, each deployed Pod automatically retrieves two IP addresses (IPv4 and IPv6). To enable IPv6 for service instances, add `IPv6` in the  **ipFamilies** field.



### Procedure

When creating a service instance of type `LoadBalancer` in Kubernetes, add the following annotation to the service to activate the dual-stack support in the AWS load balancer:

```
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-ip-address-type: dualstack
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
    service.beta.kubernetes.io/aws-load-balancer-nlb-target-type: instance
    service.beta.kubernetes.io/aws-load-balancer-type: external
...
spec:
  ipFamilyPolicy: RequireDualStack
  ipFamilies:
    - IPv6
    - IPv4
...
```

> [!TIP]
> The AWS load balancer supports the translation of data packages between IPv4 and IPv6 in both directions. If your Kyma runtime is hosted on AWS but supports only IPv4, the load balancer can handle IPv6 communication between the Internet and your Kyma runtime. Create the service as described above to retrieve a load balancer that supports both IPv4 and IPv6.

### Google Cloud

To create a dual-stack load balancer in Google Cloud, use the following configuration in the service manifest:

```
apiVersion: v1
kind: Service
...
spec:
  ipFamilyPolicy: RequireDualStack
  ipFamilies:
    - IPv6
    - IPv4
...
```

> [!WARNING]
The Kyma Istio module currently does not support the dual stack mode. Any traffic sent or received through the IPv6 network won't be part of the Istio service mesh. Consequently, Istio won't protect any IPv6 traffic, neither by transparent encryption nor by offering additional security mechanisms (for example, authentication).

## Related Information
* [Gardener documentation: Creating an IPv4/6 (dual-stack) Ingress](https://gardener.cloud/docs/guides/networking/dual-stack-ipv4-ipv6-ingress-aws/#creating-an-ipv4-ipv6-dual-stack-ingress)

