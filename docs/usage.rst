Usage
=====

*Last-Resort Gateway* uses a subcommand system. Each subcommand comes with its own
set of options. It is possible to get extra help using ``last-resort-gateway help``.

version
-------

``last-resort-gateway version`` prints the version.

daemon
------

``last-resort-gateway daemon`` run *Last-Resort Gateway* as daemon. It
should be provided the configuration file as first argument. If a TTY
is detected, logging to console is enabled automatically even if the
configuration file says otherwise.

If ``--check`` is provided as an option, the configuration file will
be checked for syntax. The process will exit with status 0 in case of
success or 1 in case of failure.

Due to the way it works, there is no way to reload its configuration
file. Just restart the daemon. The currently configured gateway are
left untouched and detected again on start. Removing gateways from the
configuration file currently leaves them too.

Currently, there is no way to interact with the daemons. ``ip route
list table 0 proto 254`` can be used to get the installed routes.
