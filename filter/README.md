# autobahnkreuz - filtering

autobahnkreuz provides a super advanced filtering since version 0.9 which is intended to solve real problems.

Imagine all your clients have subscribed to a single topic, but you want to
publish sensible information to just a few clients. On the other hand, you want
system callers to keep track of all information, so how do you solve this?

In most WAMP routers, you would either send multiple publications with the same
content but different filters, or you create multiple topics. Both variants are
not good to maintain, so we created an advanced filtering framework that lets
you combine filters.

## Warning

This feature is intended for advanced users, allows users to create "arbitrarily" complex filters which may slow down the router!

Altough our focus is on performance, reliability and security, until now this feature is considered `BETA`.

### Writing good filters

To ensure your filters perform as good as possible, write the sub-filters that
likely match (for match `any`) or likely mismatch (for match `all`) first. This
way, we have not to check ALL sub filters, so we can return early.

## Reference

Filtering is done via publish_options, which can be specified in all conformant
WAMP clients.
You can specify a 'standard' WAMP publish filter to have `eligible_authrole` and
so on, but you can also make use of the advanced filtering.

To use advanced filtering, set another key within the options dictionary: `filter_type`. Currently, `filter_type` can take 3 values:

- `not` - match if the specified filter does not match.
- `all` - all specified filters have to match
- `any` - one of the specified filters has to match

### Example - `not` - excluding a specific session

If you are having a simple filter, you can just specify it inline:
```js
const options = {
  "filter_type": "not",
  "eligible": session_id
}
```

For more complex filters, the `filter` property is used:
```js
const options = {
  "filter_type": "not",
  "filter": {
    "filter_type": "any",
    "filters": [
      {
        "eligible": session_id
      },
      {
        "blacklist_authrole": "admin"
      }
    ]
  }
}
```

### Example - `any` - matching different authroles and authids

A `filters` (plural!) property is introduced to allow multiple sub-filters to be specified:
```js
const options = {
  "filter_type": "any",
  "filters": [ // note the plural
    {
      "eligible": session_id
    },
    {
      "eligible_authrole": "admin"
    }
  ]
}
```

### Example - `all` - excluding a session when it has no specific authrole

```js
const options = {
  "filter_type": "all",
  "filters": [ // note the plural
    {
      "blacklist_authrole": "anonymous"
    },
    {
      "eligible": session_id
    }
  ]
}
```
