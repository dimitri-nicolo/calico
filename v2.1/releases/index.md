---
title: Release notes
---

{% for release in site.data.versions[page.version] %}
## {{site.prodname}} v2.1

{% include {{page.version}}/release-notes/v2.1.0-release-notes.md %}

{% endfor %}
