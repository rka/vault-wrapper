FROM node:14 as frontend-build
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build
RUN ls -la build  # List build directory contents

FROM node:14
WORKDIR /app
COPY backend/package*.json ./
RUN npm install
COPY backend/ ./
COPY --from=frontend-build /app/frontend/build ./public
RUN ls -la public  # Verify files are copied

EXPOSE 3001
CMD ["node", "server.js"]