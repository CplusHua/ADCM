#! /bin/sh

#***********************************************************************
#
# use updhistory.sh to modify /app/updhistory file
#
# Usage: updhistory.sh packagename [updatetime]
#
#***********************************************************************

#if [ "$#" -ne 2 ] ; then
#	echo "Param error"
#	exit 1
#fi

#ssl保存在"/",因为有些SSL升级包会清理/app/
if [ -L /config/etc ]; then
	UPDHISTORY="/app/updhistory"
	UPDHISTORY_TMP="/app/updhistory.tmp"
else
	UPDHISTORY="/updhistory"
	UPDHISTORY_TMP="/updhistory.tmp"
fi

if [ ! -e ${UPDHISTORY} ]; then
	touch ${UPDHISTORY}
fi

#test if exist 'head' command, if not exist, use 'busybox head'
ls /proc | head -n 1
if [ $? -eq 0 ]; then
	HEAD='head'
else
	HEAD='busybox head'
fi

if [ -f /S5100420 ]; then
	MAX_SIZE=20000
else
	MAX_SIZE=100000
fi


#if file size bigger than MAX_SIZE, delete the olderset record
FILESIZE=`ls -l ${UPDHISTORY} | awk '{ print $5}'`
while [ $FILESIZE -ge ${MAX_SIZE} ]
do
   FIRST_BLANK=`awk '/^$/ {print NR}' ${UPDHISTORY} | $HEAD -n 1`;   #get the first blank row 's row number
   if [ ! -z $FIRST_BLANK ]; then
   		sed "1,$FIRST_BLANK d" ${UPDHISTORY} > ${UPDHISTORY_TMP};	#cut row 1 ~ row $FIRST_BLANK
   		mv ${UPDHISTORY_TMP} ${UPDHISTORY}
   fi 
   FILESIZE=`ls -l ${UPDHISTORY} | awk '{ print $5 }'`;
done


echo -n "update time: " >> ${UPDHISTORY}
if [ "$#" -eq 2 ]; then
	echo $2 >> ${UPDHISTORY}
else
	date >> ${UPDHISTORY}
fi

echo -n "package name: " >> ${UPDHISTORY}
echo $1 >> ${UPDHISTORY}

cat /app/appversion >> ${UPDHISTORY}

echo "" >> ${UPDHISTORY}

sync