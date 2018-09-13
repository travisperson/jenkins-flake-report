# Jenkins Flake Report

### Install

```
go get github.com/travisperson/jenkins-flake-report
```

### Usage

Add `-render=false` to disable html rendering, and render json instead.

Defaults to `index.html` or `index.json`, in the current directory, use `-output` to specify a different file.

```
jenkins-flake-report -project js-ipfs -branch master -start 150 -end 237
```

### Report Cache

Report files are cached under `$HOME/.testchart`
