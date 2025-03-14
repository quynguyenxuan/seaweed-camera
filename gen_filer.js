const fs = require('fs');
const { exec } = require('child_process');

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
const packages = [
    "001",
    "003",
    "007",
    "015",
    "030",
]
const data = {
  
};

const locations = []
for (const bucket of buckets) {
    for (const pkg of packages) {
        locations.push({
            locationPrefix: `/buckets/${bucket}/${pkg}`,
            collection: `${bucket}_${pkg}`,
        })
    }
}
data.locations = locations;
const jsonData = JSON.stringify(data, null, 2);

fs.writeFile('filer.conf', jsonData, (err) => {
  if (err) {
    console.error('Error writing file:', err);
  } else {
    console.log('JSON data written to filer.json');
    setTimeout(() => {
        exec('curl -F file=@/mnt/nvme0n1p6/seaweedfs385/filer.conf "http://localhost:8888/etc/seaweedfs/"', (error, stdout, stderr) => {
            if (error) {
              console.error(`exec error: ${error}`);
              return;
            }
            console.log(`stdout: ${stdout}`);
            console.error(`stderr: ${stderr}`);
          });    
    }, 2000);
    
  }
});


