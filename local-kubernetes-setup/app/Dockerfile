FROM nginx:stable-alpine
ARG BUILD_ID=dev
ARG BUILD_DATE=unknown
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY index.html.template /tmp/index.html.template
RUN sed -e "s/{buildnumber}/${BUILD_ID}/g" \
        -e "s/{builddate}/${BUILD_DATE}/g" \
        /tmp/index.html.template > /usr/share/nginx/html/index.html \
 && touch /usr/share/nginx/html/favicon.ico
