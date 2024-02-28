FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-calendly"]
COPY baton-calendly /