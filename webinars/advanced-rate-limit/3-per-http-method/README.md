# Per HTTP Method

Now let's try to create several limits, this time utilizing the Method pseudo-header. This is related to a use 
case some call "load shedding". 

We can patch settings with this:

`k patch -n gloo-system settings default --type merge --patch "$(cat settings-patch.yaml)"`

And apply the rate limit actions to our route:

`k apply -f vs.yaml`


Now if we run this a few times we should see rate limiting:
`curl $(glooctl proxy url)/ -v`

But we can see other HTTP methods have a larger limit:

`curl $(glooctl proxy url)/ -v -XPOST`

