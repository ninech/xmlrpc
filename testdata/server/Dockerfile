FROM ruby:3.3-slim

RUN gem install xmlrpc

WORKDIR /app
COPY test_server.rb .

EXPOSE 5001

CMD ["ruby", "test_server.rb"]
