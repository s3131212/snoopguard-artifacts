FROM debian:bookworm
WORKDIR /root
RUN apt update
RUN apt install -y wget
RUN wget "https://cdn.geekbench.com/Geekbench-6.3.0-LinuxARMPreview.tar.gz"
RUN tar -xzf Geekbench-6.3.0-LinuxARMPreview.tar.gz
CMD /root/Geekbench-6.3.0-LinuxARMPreview/geekbench_aarch64