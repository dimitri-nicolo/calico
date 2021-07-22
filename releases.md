---
title: Calico Enterprise Documentation Archives
description: Home
layout: docwithnav
---
This page contains permalinks to specific versions of {{site.prodname}} documentation, as well as links to the latest released
and the nightly build of documentation. Each set of versioned docs includes a Release Nodes page for that particular
version.
{%- if site.archive %}
- [latest](/) (currently {{site.data.versions.first.title}})
- [nightly](/master/){: data-proofer-ignore=""} (master)
- [{{site.data.versions.first.title}}](/{{page.version}})
{%- for version in site.data.archives %}
{%- if version.first %}
    {%- for v in version["legacy"] %}
- [{{ v }}](/{{ v }}/){: data-proofer-ignore=""}
    {%- endfor %}
{%- else %}
- [{{ version }}](/{{ version }}/)
{%- endif %}
{%- endfor %}

<div id="release-list" class="hidden" markdown="0">
    <li><a href="/">latest</a></li>
    <li role="separator" class="divider"></li>
    <li><a href="/master">nightly</a></li>
    <li><a href="/{{page.version}}">{{site.data.versions.first.title}}</a></li>
    {%- for version in site.data.archives %}
        {%- if version.first %}
        {%- for v in version["legacy"] %}
        <li><a href="/{{ v }}">Version {{ v | replace: "v", "" }}</a></li>
        {%- endfor %}
        {%- else %}
        <li><a href="/{{ version }}">Version {{ version | replace: "v", ""  }}</a></li>
        {%- endif %}
    {%- endfor %}
</div>
{% endif %}
