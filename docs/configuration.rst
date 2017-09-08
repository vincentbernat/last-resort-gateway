Configuration
=============

*Last-Resort Gateway* is configured through a YAML file that should be
provided to the daemon subcommand on start. Each aspect of the daemon
is configured in a different section:

 - ``reporting``: `Reporting`_
 - ``gateways``: `Gateways`_

Currently, due to a technical limitation, if a key is unknown, it will
just be ignored. Be extra-careful that all keys are correctly spelled.

Reporting
---------

Reporting encompasses the following aspects, each of them having its
own subsection:

 - ``logging``: `Logging`_,
 - ``metrics``: `Metrics`_,
 - ``sentry``: `Crash reporting`_.

Logging
~~~~~~~

Here is an example configuration:

.. code-block:: yaml

    reporting:
      logging:
        level: info
        console: true
        syslog: false
        files:
          - /var/log/last-resort-gateway.log
          - json:/var/log/last-resort-gateway.log

``level`` specify the maximum log level to use. It can be one of:

 - ``crit``
 - ``error``
 - ``warn``
 - ``info``
 - ``debug``

``console`` enables logging to console while ``syslog`` enables
logging to the local syslog daemon. No configuration knobs are
available for those targets.

``files`` allows one to set a list of files to log to. It's possible
to prefix the file with the expected format. Currently, only ``json:``
is allowed to get JSON event format.

Metrics
~~~~~~~

Metrics can be exported using various output plugins. Here is an example:

.. code-block:: yaml

    reporting:
      metrics:
        - expvar:
            listen: 127.0.0.1:8123

The support outputs are:

 - ``expvar``
 - ``file``
 - ``collectd``

The ``expvar`` output supports the following key:

 - ``listen`` to specify an HTTP endpoint to listen to (mandatory)

The ``file`` output supports the following keys:

 - ``path`` to specify a file path (mandatory)
 - ``interval`` to specify an interval (mandatory)

At each tick, the current metric values will be written to the
specified file as a one-line JSON object. For debug purpose, it's
possible to filter the metrics concerning only *Last-Resort Gateway*
by using the following command::

    tailf /var/log/last-resort-gateway/metrics \
      | jq 'with_entries(select(.key | startswith("jura.")))'

The ``collectd`` output supports the following keys:

 - ``connect`` to specify the target (default to ``127.0.0.1:25826``)
 - ``interval`` to specify an interval (mandatory)

For collectd output to work correctly, you need to append the
following to ``types.db`` file::

     histogram count:COUNTER:0:U, max:GAUGE:U:U, mean:GAUGE:U:U, min:GAUGE:U:U, stddev:GAUGE:0:U, p50:GAUGE:U:U, p75:GAUGE:U:U, p95:GAUGE:U:U, p98:GAUGE:U:U, p99:GAUGE:U:U, p999:GAUGE:U:U
     meter     count:COUNTER:0:U, m1_rate:GAUGE:0:U, m5_rate:GAUGE:0:U, m15_rate:GAUGE:0:U, mean_rate:GAUGE:0:U
     timer     max:GAUGE:U:U, mean:GAUGE:U:U, min:GAUGE:U:U, stddev:GAUGE:0:U, p50:GAUGE:U:U, p75:GAUGE:U:U, p95:GAUGE:U:U, p98:GAUGE:U:U, p99:GAUGE:U:U, p999:GAUGE:U:U

Note that the configuration should be a list of output plugins. An
output plugin is a map from plugin type to its configuration. Only one
item per map is allowed.

Intervals are specified with a number and a unit. For example:

 - ``5s``
 - ``1m``
 - ``30m``

Crash reporting
~~~~~~~~~~~~~~~

Crash reporting is done with Sentry. Here is an example configuration:

.. code-block:: yaml

    reporting:
      sentry:
        dsn: https://public:secret@sentry.example.com/last-resort-gateway
        tags:
          environment: production

Gateways
--------

This is the most important part of the configuration. It contains a
list of last resort gateway to maintain. Each element of the list
describes a gateway.

.. code-block:: yaml

    gateways:
      - from:
          prefix: 0.0.0.0/0
          protocol: bird
          table: public
        to:
          protocol: 254
          metric: 4294967295
          blackhole: yes
      - from:
          prefix: ::/0
          protocol: bird
          table: public
        to:
          protocol: 254
          metric: 4294967295
          blackhole: yes

The above configuration will maintain a last resort default gateway
for both IPv4 and IPv6. Each gateway contains a ``from`` block and a
``to`` block. Only the ``from`` block is mandatory.

Also note, it is not a good idea to have collisions between the gateways.

From block
~~~~~~~~~~

The ``from`` block selects a route to be used to build the last resort
gateway. It contains the criteria the route should match. If several
routes match, the lowest metric wins.

 - ``prefix``. Mandatory. Prefix of the route entry. Most of the time,
   this should be the default route.
 - ``protocol``. Optional. Protocol of the route entry. Can be a
   number (between 0 and 255) or a name. Names are looked up in
   ``/etc/iproute2/rt_protos`` and
   ``/etc/iproute2/rt_protos.d/*.conf``.
 - ``metric``. Optional. Metric of the route entry.
 - ``table``. Optional. Table of the route entry. Can be a number
   (between 0 and 255) or a name. Names are looked up in
   ``/etc/iproute2/rt_tables`` and
   ``/etc/iproute2/rt_tables.d/*.conf``. By default, the main table is
   used.

To block
~~~~~~~~

The ``to`` block instructs how the selected route should be
transformed as a last resort gateway. The criteria expressed in this
section are also used on start to find if a last resort route from a
previous run is already here. All keys are optional.

 - ``prefix``. Prefix for the last resort gateway. By default, this is
   the same prefix as the selected route. It should be of the same
   family as the prefix of the selected route.
 - ``protocol``. Protocol of the last resort gateway. By default, this is 254.
 - ``metric``. Metric of the last resort gateway. By default, this is
   4294967295 (the maximum possible metric). The idea is to use the
   highest possible metrics to not shadow a valid gateway.
 - ``table``. Table of the last resort gateway. By default, this is
   the same table as the selected route.
 - ``blackhole``. If true, a blackhole route will be used as a last
   resort route if we no route in the ``from`` block can be selected
   and we don't have a last resort route already installed. This is
   useful only on the first start of the daemon if we want to ensure
   trafic doesn't escape a routing table until routing daemons are
   able to install routes.
