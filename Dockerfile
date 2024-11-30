# Build stage for React frontend
FROM node:14 as frontend-build
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Build stage for Node.js backend
FROM node:14
WORKDIR /app
COPY backend/package*.json ./
RUN npm install
COPY backend/ ./
COPY --from=frontend-build /app/frontend/build ./public

EXPOSE 3001
CMD ["node", "server.js"]