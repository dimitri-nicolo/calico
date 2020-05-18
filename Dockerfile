FROM alpine:3 as builder
COPY kibana /kibana
RUN apk add --no-cache zip
RUN zip -r /tigera_customization.zip kibana

FROM docker.elastic.co/kibana/kibana:7.6.0

USER root
RUN yum -y update && yum -y upgrade
# The base "kibana" user has a uid of 1000
USER 1000

# custom throbber
RUN sed -i 's/Loading Kibana/Loading Tigera/g' /usr/share/kibana/src/core/server/rendering/views/template.js

# custom HTML title information
RUN sed -i 's/title Kibana/title Tigera/g' /usr/share/kibana/src/legacy/server/views/index.pug

# custom plugin css
COPY --from=builder /tigera_customization.zip /
RUN sed -i 's/reverse()/reverse(),`${regularBundlePath}\/tigera_customization.style.css`/g' /usr/share/kibana/src/legacy/ui/ui_render/ui_render_mixin.js

# Modify logoKibana in vendorsDynamicDLL to be empty. Custom icon will be set as background-image in gradiant_style plugin css
RUN sed -i 's@evenodd"}.*)))},@evenodd"}))},@g' /usr/share/kibana/node_modules/@kbn/ui-shared-deps/target/icon.logo_kibana-js.js

RUN bin/kibana-plugin install file:///tigera_customization.zip
RUN bin/kibana --env.name=production --logging.json=false --optimize
