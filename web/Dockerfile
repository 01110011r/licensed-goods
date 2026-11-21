# Web image: Vite dev server by default. The test-web service overrides the
# command to run lint and the production build.
FROM oven/bun:1

WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .

EXPOSE 5173
CMD ["bun", "run", "dev", "--", "--host", "0.0.0.0", "--port", "5173"]
