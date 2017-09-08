Last-Resort Gateway documentation
=================================

*Last-Resort Gateway* maintains last resort routes in the absence of
any routing daemon. It is Linux-only (no abstraction is in place to
support other OSes).

When a route for a given table is updated, a copy of it is made with a
higher metric to serve as a last resort route in case the original
route disappeared. This route is kept up-to-date except if the
original route completely disappears. In this case, the last known
route is kept.

The idea is to not disrupt the traffic in case of a transient failure
of a routing daemon (for example, an upgrade). The last resort gateway
should be able to handle all routes, even if this is not the best
path.

.. toctree::
   :maxdepth: 2

   install
   configuration
   usage
   integration

Indices and tables
==================

* :ref:`genindex`
* :ref:`search`

