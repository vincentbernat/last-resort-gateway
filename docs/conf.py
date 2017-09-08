# -*- coding: utf-8 -*-

import os

source_suffix = '.rst'
master_doc = 'index'

# General information about the project.
project = u'Last-Resort Gateway'
copyright = u'2017, Exoscale'
version = os.environ.get('VERSION', '0')
release = version

exclude_patterns = ['_build']
pygments_style = 'sphinx'

try:
    import alabaster
    html_theme = 'alabaster'
    html_theme_options = {
        "font_family": "serif"
    }
    if alabaster.version.__version_info__ >= (0, 7, 8):
        html_theme_options["fixed_sidebar"] = True
except ImportError:
    pass
