# components/Dockerfile
FROM nginx:mainline-alpine3.18

COPY ./index.html /usr/share/nginx/html/index.html

CMD ["nginx", "-g", "daemon off;"]