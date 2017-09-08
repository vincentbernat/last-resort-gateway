Integration
===========

*Last-Resort Gateway* expects the selected routes to be installed by
 some processes, most notably a routing daemon. In this case, it is
 important to configure the routing daemon to ignore routes installed
 by *Last-Resort Gateway*.

BIRD integration
----------------

The easiest way to make *BIRD* ignore the routes installed by *Last
Resort Gateway* is to modify the ``kernel`` protocol definition if the
``learn`` flag is enabled::

    template kernel {
      persist;
      learn;
      import where krt_source != 254;
      export all;
      merge paths yes;
      scan time 10;
    }

