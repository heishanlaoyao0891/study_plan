FROM node:20-alpine AS build

ARG VITE_API_BASE=/api
ENV VITE_API_BASE=${VITE_API_BASE}

WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --registry=https://registry.npmmirror.com

COPY frontend/ ./
RUN npm run build:h5

FROM nginx:1.27-alpine

COPY deploy/nginx/frontend.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist/build/h5 /usr/share/nginx/html
EXPOSE 80
