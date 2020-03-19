In the simple case, we can have:
* Gloo deployed to an admin namespace (gloo-system) that is locked down
* Gloo watching all namespaces
* Gloo discovery is disabled
* A dev has the ability to create upstream, virtual service, along with other application resources in a specific namespace

Two limitations of this approach:
* Hard to prevent one team's config from interfering with another 
  * For example, don't want another team creating a VS with the same domain
* Hard to enforce common config at the vhost level / across all routes
