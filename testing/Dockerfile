FROM  postman/newman:alpine

ENV APP_HOME=/home/testing
RUN mkdir -p $APP_HOME
WORKDIR $APP_HOME

ADD . $APP_HOME
RUN chmod +x $APP_HOME/entrypoint.sh