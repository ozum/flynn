FROM flynn/cedarish

ADD Gemfile /app/Gemfile
ADD Gemfile.lock /app/Gemfile.lock

# install Go
RUN wget https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz
RUN tar -C /usr/local -zxvf go1.4.2.linux-amd64.tar.gz
ENV PATH /usr/local/go/bin:$PATH
ENV GOPATH /gopath

RUN go get github.com/jvatic/asset-matrix-go

RUN curl -sL https://deb.nodesource.com/setup | bash -

RUN apt-get update &&\
    apt-get install -y nodejs build-essential &&\
    gem install bundler --no-rdoc --no-ri &&\
    cd /app &&\
    bundle install --deployment &&\
    chmod -R o+rw /app
