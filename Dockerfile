FROM alpine:3.7 as builder
COPY kibana /kibana
RUN apk add --no-cache zip
RUN zip -r /tigera_customization.zip kibana

FROM docker.elastic.co/kibana/kibana:6.4.1
# custom favicons
COPY favicons/* /usr/share/kibana/src/ui/public/assets/favicons/

# custom plugin css
COPY --from=builder /tigera_customization.zip /
RUN sed -i "s/commons.style.css'),/commons.style.css'),createAnchor('{{bundlePath}}\/tigera_customization.style.css'),/g" /usr/share/kibana/src/ui/ui_render/bootstrap/template.js.hbs
RUN bin/kibana-plugin install file:///tigera_customization.zip
