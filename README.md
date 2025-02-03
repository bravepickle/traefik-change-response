# traefik-change-response
Traefik Proxy plugin to change downstream response based on returned HTTP status code

### Install
1. Run `make setup` - will init environment and dependencies for testing application
2. Run `make test` - run tests to ensure everything works as expected
3. Run `make bench` - run benchmark tests to check performance
4. See [Traefik tutorials](https://plugins.traefik.io/install) on how to install & use plugins

### Configuration
Below is the example for plugin configuration and explanations. This setup will convert all 500, 501 HTTP status codes
to 200, remove & set some headers in the response before returning it to the client
```yaml
  debug: false # in debug mode some additional debug messages & headers will be returned to the Traefik application
  # list of override rules - at least one should be defined
  overrides:
    - from: [500, 501] # list of initial downstream response codes (returned from the backend server) to match against the rule for processing
      to: 200          # HTTP status code to replace initial ones 
      body: ""         # response body in string format to set for the rule 
      mode: replace    # override mode to use. Available: 
                       #   - replace (default) - will replace existing response body with the provided contents. Static content
                       #   - keep - will ignore custom "body" value and keep the response body as it is. Headers and status code may be affected
                       #   - append - will append to the response body some extra content
                       #   - prepend - will prepend before the response body some extra content
      removeHeaders: [Content-Encoding, Transfer-Encoding] # will remove the provided headers from downstream response
      headers:         # will set/add/overwrite response headers before sending to the client. Content-Length will be ignored
        X-Overridden: [Yes]
        
    # this is chaining rule that will add extra headers only for 501 status code responses 
    - from: [501]      # it will look for the initial response code, not the replaced one by the previous rule.
                       # If there is a need to chain status codes and take into account replaced status codes one can
                       # setup additional middleware for the same plugin, but with different configuration
      to: 200          # we want to always return 200 status code
      mode: keep       # keep the body as it is
      headers:
        X-Foo: [bar]   # set additional headers
```

### TODOs
- [ ] consider how better to handle `Transfer-Encoding: chunked` data and automatically fix issues with incorrect response processing. E.g. `Content-Length`
