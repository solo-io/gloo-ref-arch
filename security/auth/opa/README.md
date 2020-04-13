# Rapid Iteration on OPA Policies with Enterprise Gloo

Enterprise Gloo has long supported an integration with Open Policy Agent, allowing users to define policies in OPA's 
**Rego** language and apply them on routes through the Gloo API Gateway with zero code or deployment overhead required. 

In Gloo Enterprise 1.3.0, we upgraded to OPA 0.18.0 and made a few logging improvements that enable a simple and 
rapid-iteration workflow for testing OPA policies on your routes. 

