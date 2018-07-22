## Feature Authorizer

### Features

Features is a theoretical construct for grouping functions.
Any number of URIs can be grouped into a feature, called FeatureMapping.
The permissions to a feature can be defined per authrole, this is called FeatureMatrix.

### FeatureMatrix

A FeatureMatrix should be in the following structure:

```json
{
    "rocks.test.admin":
    {
        "admin": true,
        "user": false
    },
    "rocks.test.tickets.manage":
    {
        "admin": true,
        "user": true
    }
}
```

It has to be returned by an user-provided (WAMP RPC) endpoint as first argument. This endpoint has to be provied to `--feature-authorizer-matrix-func`, equally to `--authorizer-func`.

### FeatureMapping

A FeatureMapping should be in the following structure:

```json
{
    "rocks.test.admin" : [
        "rocks.test.user.create",
        "rocks.test.user.delete"
    ],
    "rocks.test.tickets.manage": [
        "rocks.test.ticket.createticket",
        "rocks.test.ticket.sendmail"
    ]
}
```

`rocks.test.user.create` is an URI from an endpoint. Until now, there is no difference between publish, subscribe, register and call.

### Updating FeatureMatrix

Initially, the Feature Authorization is not setup correctly. You have to call `wamp.featureauth.update` to let the router fetch the neweset values from your provided endpoints. This can be done as many times as you want, but you have to do it manually after a change in the generation of the feature matrix and mapping.

If you enable Feature Authorization, you have to make sure, that the service, which registers the functions and calls the first update, has an trusted authrole, which can be provided via `--trusted-authroles`
