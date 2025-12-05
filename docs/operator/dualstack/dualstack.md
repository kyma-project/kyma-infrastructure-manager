# Kyma Runtime with Dual-Stack Support

On the Google Cloud and Amazon Web Services IaaS providers, SAP BTP, Kyma runtime can support both Internet Protocol (IP) versions IPv4 and IPv6 in parallel (dual stack).

To retrieve support for both Internet protocols, create your Kyma runtime instance with dual-stack support enabled.

This feature is only available for newly created Kyma runtime instances. For existing runtimes, adding support for IPv6 afterwards is not possible, so they only use IPv4. The exception is single-stack clusters running on Amazon Web Services. For these clusters, you can enable support for public IPv6 traffic. See the hint in the [External Load Balancer: AWS](#amazon-web-services) section.

When you create a Kyma cluster with dual-stack support, each deployed Pod automatically retrieves two IP addresses (IPv4 and IPv6). To enable IPv6 for service instances, add `IPv6` in the  **ipFamilies** field.

> [!WARNING]
> The Kyma Istio module currently does not support dual-stack mode. Any traffic sent or received through the IPv6 network is not part of the Istio service mesh. Consequently, the Istio module does not protect IPv6 traffic, either by transparent encryption or by offering additional security mechanisms (for example, authentication).

### Procedure

Create a service instance of type `LoadBalancer` in your Kyma cluster. In the service manifest, add the **ipFamilyPolicy** field with the value `RequireDualStack`, and add the value `IPv6` in the **ipFamilies** field.

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
In your Kyma cluster with dual-stack support, each deployed Pod automatically receives two IP addresses: IPv4 and IPv6.

## Related Information
* [Gardener documentation: Creating an IPv4/6 (dual-stack) Ingress](https://gardener.cloud/docs/guides/networking/dual-stack-ipv4-ipv6-ingress-aws/#creating-an-ipv4-ipv6-dual-stack-ingress)

