FROM node:24-alpine
WORKDIR /app
COPY package.json server.js ./
ENV PORT=3000
EXPOSE 3000
USER node
CMD ["node", "server.js"]
