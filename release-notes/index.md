---
title: Release notes
description: What's new, and why features provide value for upgrading.
canonical_url: '/release-notes/'
---
<div class="git-hash" id="{{site.data['hash']}}">
</div>

The following table shows component versioning for {{site.prodname}}  **{{ page.version }}**.


To select a different version, click **Releases** in the top-right navigation bar.

{% for release in site.data.versions %}
## Calico Enterprise {{ release.title }}

{% if release.note %}
{{ release.note }}
{% else %}
{% include release-notes/{{release.title}}-release-notes.md %}
{% endif %}

## Component Versions

| Component              | Version |
|------------------------|---------|
{% for component in release.components %}
{%- capture component_name %}{{ component[0] }}{% endcapture -%}
| {{ component_name }}   | {{ component[1].version }} |
{% endfor %}

{% endfor %}
