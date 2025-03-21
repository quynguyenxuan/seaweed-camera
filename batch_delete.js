const fs = require('fs');
const { exec } = require('child_process');
const url = require('url');

const buckets = [
    "camera2009",
    "camera2010",
    "camera2011",
    "camera2012",
    "camera2013",
    "camera2014",
    "camera2015",
    "camera2016",
    "camera2017",
    "camera2018",
    "camera2019",
]
const packages = {
    1: "001",
    // 3: "003",
    // 7: "007",
    // 15: "015",
    // 30: "030",
}
process.env.TZ="Asia/Ho_Chi_Minh"
const filerUrl = "http://localhost:8888/buckets"
function getEndOfDaysAgo(now, day) {
  const sevenDaysAgo = new Date(now);
  sevenDaysAgo.setDate(now.getDate() - day);
  // Set the time to the end of the day (23:59:59.999)
  sevenDaysAgo.setHours(23, 59, 59, 999);

  return sevenDaysAgo;
}
const now = new Date();

for (const bucket of buckets) {
    for (const [daysAgo,pkg] of Object.entries(packages)) {
      const fullUrl = url.format({
        pathname: `${filerUrl}/${bucket}/${pkg}`,
        query: {
          collection: `${bucket}_${pkg}`,
          fromTime: 1732953740,
          toTime: +Math.round(+getEndOfDaysAgo(now, daysAgo) / 1000),
        },
      });
      console.log(`Deleting ${fullUrl}`);
      // fetch(fullUrl, {
      //   method: "DELETE",
      //   headers: {
      //     "Content-Type": "application/json",
      //   },
        
      // }).then((response) => {
      //   if (response.ok) {
      //     console.log(`Deleted ${fullUrl}`);
      //   } else {
      //     console.error(`Failed to delete ${fullUrl}: ${response.status}`);
      //   }
      // });
     
    }
}


