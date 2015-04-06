FROM google/golang
WORKDIR /gopath/src/github.com/hashicorp/terraform
RUN apt-get update && apt-get install -y wget build-essential ruby-dev git-core mercurial
RUN gem install fpm
ADD . /gopath/src/github.com/hashicorp/terraform
RUN make updatedeps && make
CMD ["/gopath/src/github.com/hashicorp/terraform/scripts/docker_installer.sh"]

