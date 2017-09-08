Installation
============

Compilation
-----------

*Last-Resort Gateway* is a self-contained executable. You need a
proper installation of *Go*. Building is then pretty simple::

    make

You'll either get a cryptic error or an executable in ``bin/last-resort-gateway``.

Debian packages
---------------

Alternatively, you can get some Debian packages by using the following command::

    dpkg-buildpackage -us -uc -b

The Debian package will come with the ``last-resort-gateway``
executable but also with the appropriate *systemd* unit. It is
expected to work with the following distributions (if you are able to
compile it):

 - Debian 8 or more recent,
 - Ubuntu Precise or more recent.
