---
title: Release notes
redirect_from: latest/release-notes/index
canonical_url: 'https://docs.projectcalico.org/v3.7/release-notes/index'
---
<div class="git-hash" id="{{site.data['hash']}}">
</div>

The following table shows component versioning for {{site.prodname}}  **{{ page.version }}**.

Use the version selector at the top-right of this page to view a different release.

{% for release in site.data.versions[page.version] %}
## Calico Enterprise {{ release.title }}

{% if release.note %}
{{ release.note }}
{% else %}
{% include {{page.version}}/release-notes/{{release.title}}-release-notes.md %}
{% endif %}

## Component Versions

| Component              | Version |
|------------------------|---------|
{% for component in release.components %}
{%- capture component_name %}{{ component[0] }}{% endcapture -%}
| {{ component_name }}   | {{ component[1].version }} |
{% endfor %}

{% endfor %}
