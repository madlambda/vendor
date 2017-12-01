# Vendor: Go vendoring tool

This project started as an experiment to check if a tool
that does what we need could be developed fast and be
really small, in the end it seems like a success (at
least for now).

What is the magic behind being small ? Our dependency
management needs are much smaller than other tools
tries to contemplate.

For example, there is the Go community tool
[dep](https://github.com/golang/dep) which gives support
to a lot of things that we don't use like semantic versioning,
it will create TOML and lock files etc, we don't want that
because we don't think we need that, at least in our context.

## What is our context ?

Our context is:

* Keep dependencies small (in size and number)
* Depend on reliable code
* Reliable code is continuously integrated and has tests
* If it is reliable just updating to the master should be enough
* Our code must be equally reliable
* Keeping all your dependencies update is important (bugs/security)

Since our code must be reliable we can just follow this flow:

* Vendor dependencies (updates everything with go get)
* Build code 
* Test code
* Go have beer =)

If a dependency breaks something we fix it or just rollback.
Since we don't have a lot of dependencies we usually don't need
to update one of the dependencies, we just update everything
(keeping dependencies updated is good).

This is the gist of why we built another dependency management
tool for Go, it is as lightweight as it can get.
