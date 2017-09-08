# Last-Resort Gateway

Maintain last resort routes in the absence of any routing daemon.

When a route for a given table is updated, a copy of it is made with a
higher metric to serve as a last resort route in case the original
route disappeared. This route is kept up-to-date except if the
original route completely disappears. In this case, the last known
route is kept.

## Requirements

 - golang (>= 1.8)
 - sphinx
 - make

## Build

To build the daemon:

    make last-resort-gateway

To get documentation (needs Sphinx):

    make docs

Vendoring is done with `godep`. `make` should take care of pulling the
dependencies automatically. You can update the dependencies with:

    make vendor-update PKG=github.com/kylelemons/godebug
    make vendor-update # all dependencies

Some tests need to be run into an empty network namespace. Some other
tests need escalated privileges. All this is handled automatically if
the current user has the appropriate rights to create namespaces. This
can be done when running as root or by setting the following sysctl:

    sysctl -w kernel.unprivileged_userns_clone=1

Then, you can run `make test` to run tests.

## Packages

To build Debian packages, either use:

    dpkg-buildpackage -us -uc -b

Or:

    VERSION=$(make version) gbp buildpackage \
        --git-dist=xenial \
        --git-ignore-branch --git-debian-branch=HEAD --git-upstream-tag=HEAD \
        --git-cleaner=/bin/true -nc
