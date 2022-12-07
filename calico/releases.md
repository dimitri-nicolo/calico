---
title: Calico Enterprise Documentation Archives
description: Home
layout: docwithnav
---
This page contains permalinks to specific versions of {{site.prodname}} documentation, as well as links to the latest released
and the nightly build of documentation. Each set of versioned docs includes a Release Notes page for that particular
version.

- [master](/master){: data-proofer-ignore=""} (nightly)
{%- for archive in site.data.archives %}
  {%- if archive.preview %}
- [{{ archive.version }}](/{{ archive.version }}){: data-proofer-ignore=""} (preview)
  {%- elsif archive.latest %}
    {%- if site.data.versions.first.title == "master" %}
- [latest](/{{ archive.version }}){: data-proofer-ignore=""} (currently {{archive.version}})
    {%- else %}
- [latest](/{{ archive.version }}){: data-proofer-ignore=""} (currently {{site.data.versions.first.title}})
    {%- endif %}
  {%- else %}
- [{{ archive.version }}](/{{ archive.version }}){: data-proofer-ignore=""}
  {%- endif %}
{%- endfor %}

<div id="release-list" class="hidden" markdown="0" data-proofer-ignore>
    {%- for archive in site.data.archives %}
        {% if archive.latest %}
            <li><a href="/{{archive.version}}">{{archive.version}}<span class="badge release-badge latest">latest</span></a></li>
        {% endif %}  
    {%- endfor %} 
    <li role="separator" class="divider"></li>
    <li><a href="/master">master<span class="badge release-badge nightly">nightly</span></a></li>
    {%- for archive in site.data.archives %}
        {% if archive.preview %}
            <li><a href="{{ archive.version }}">{{ archive.version }}<span class="badge release-badge preview">preview</span></a></li>
        {% else %}
            <li><a href="{{ archive.version }}">{{ archive.version }} </a></li>
        {% endif %}
        {%- if forloop.index > 5 %}{% break %}{% endif %}
    {%- endfor %} 
    <li><a href="/releases">Earlier versions</a></li>
</div>
