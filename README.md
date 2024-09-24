# 本地构建tvbox直播源

支持在Windows和Linux构建。

如需检测链接有效性，可在下方网站下载对应平台的ffprobe可执行文件。

https://ffbinaries.com/downloads

### 【直接运行】
源程序默认按照目录下的config.json文件作为配置文件，运行前将config.json文件与程序放置在同一目录下。

同时只输出ipv6链接并检测其有效性，运行前将ffprobe可执行文件与程序放置在同一目录下。

在 windows平台的cmd命令行中执行exe文件
````bash
.\iptv-collection.exe
````
在liunx平台的终端中执行二进制文件
````bash
./iptv-collection
````

### 【可选参数配置】
-config:自定义配置文件路径，你可以在config.json文件中的url参数中配置上游直播源链接，支持m3u和txt文件，在categoryList参数中配置输出的类别和节目名格式。
````bash
./iptv-collection -config=config.json
````
-type: 用于筛选链接的类型，支持ipv4、ipv6、all三种参数。ipv4只生成live_v4.txt文件，ipv6只生成live_v6.txt文件，all则两个文件都生成。
````bash
./iptv-collection -type=ipv6
````
-check: 用于指定是否检测链接的有效性，支持yes和no。源数量和有效性检测会导致运行时间较长，不建议在ipv4链接中使用。
````angular2html
./iptv-collection -check=yes
````
### 【自定义构建】
````bash
go build iptv-collection
````