#!/bin/sh

LOGF="/tmp/updatellog.txt"
DEBUG_LOG="/var/appsh1_debug.log"          #for delete
ERR_MSG="/var/upd_sh_err.log"

date >$LOGF
echo "appsh start at `date`" >>$DEBUG_LOG   #for delete

grep -q "mother_1.0" /app/appversion
if [ $? -eq 0 ]
then
    echo "不支持从当前母盘升级" > $ERR_MSG
    exit 1
fi
# 只能从AD-4.8 AD-4.9升级
if ! grep -q "AD-5.[34]" /app/appversion
then
    echo "不能从当前版本升级" > $ERR_MSG
    exit 1
fi

# 判断是Sinfor-M5x00-AD-2.0.0还是M5x00-AD1.0.0格式
head -n 1 /app/appversion | grep -q "^SANGFOR-M\|^Sinfor-M"
if [ $? -eq 0 ]
then
    ARCHCFG=`head -n 1 /app/appversion | awk -F- '{print $2}'`
else
    ARCHCFG=`head -n 1 /app/appversion | awk -F- '{print $1}'`
fi
#
DEVERSION=`head -n 1 /app/deversion`

# 检查当前升级的旧版本是否符合
# 不能从定制版升级，当前定制版只有node_pre_policy
if grep -q -i node_pre_policy /app/appversion
then
    echo "不能从定制版升级" > $ERR_MSG
    exit 1
fi

if grep -q -i custom_version /app/appversion
then
    echo "不能从定制版升级" > $ERR_MSG
    exit 1
fi

if grep -q -i Custom-built /app/appversion
then
    echo "不能从定制版升级" > $ERR_MSG
    exit 1
fi

if head -n 1 /app/appversion | grep -q EN
then
    echo "不能从英文版升级" > $ERR_MSG
    exit 1
fi

# 内存小于4G不能升级
memok=`grep MemTotal /proc/meminfo | awk '
{
    if($2>3500000)
    {
        print "ok"
    }
    else
    {
        print "notok"
    }
}'`

if [ "x$memok" == "xnotok" ]
then
    echo "硬件配置太低，请联系厂家升级硬件平台！" > $ERR_MSG
    exit 1
fi

