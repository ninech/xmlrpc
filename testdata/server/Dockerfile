FROM ruby:3-alpine

RUN gem install xmlrpc

WORKDIR /usr/local/src/xmlrpcserver
COPY *.rb .

EXPOSE 5001

CMD ["ruby", "server.rb"]
