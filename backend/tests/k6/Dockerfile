FROM grafana/k6:latest
WORKDIR /app
COPY loadtest.js .
ENTRYPOINT ["k6", "run"]
CMD ["loadtest.js"]