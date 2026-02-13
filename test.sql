version: '3.8'

services:
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    env_file:
      - ./backend/.env
    restart: unless-stopped
    networks:
      - ailaw-network

  frontend:
    build:
      context: ./frontend/ailaw-webpage
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    env_file:
      - ./frontend/ailaw-webpage/.env
    depends_on:
      - backend
    restart: unless-stopped
    networks:
      - ailaw-network

  model:
    build:
      context: ./model/FastAPI
      dockerfile: Dockerfile
    ports:
      - "8000:8000"
    env_file:
      - ./model/FastAPI/.env
    restart: unless-stopped
    networks:
      - ailaw-network

networks:
  ailaw-network:
    driver: bridge