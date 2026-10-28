package main

import (
 "context"
 "database/sql"
 "encoding/json"
 "fmt"
 "net/http"
 "os"
 "strings"
 "time"
 _ "github.com/go-sql-driver/mysql"
)

type processLogRecord struct {
 Server string `json:"server"`
 ProcessDate interface{} `json:"process_date"`
 ExchID interface{} `json:"exch_id"`
 Segment interface{} `json:"segment"`
 Process interface{} `json:"process"`
 SuccessFlag interface{} `json:"success_flag"`
 ProcessStatus interface{} `json:"process_status"`
 Success interface{} `json:"success"`
 Rejected interface{} `json:"rejected"`
 UploadFileName interface{} `json:"upload_file_name"`
 FileName interface{} `json:"file_name"`
}
type processLogError struct { Server string `json:"server"`; Error string `json:"error"` }
type processLogResponse struct { Records []processLogRecord `json:"records"`; Errors []processLogError `json:"errors"` }
type processDBConfig struct { server,host,port,name,user,pass string }

// Add registerProcessLogRoute() inside the existing rms_api.go main(), before ListenAndServe.
func registerProcessLogRoute(){ http.HandleFunc("/api/market-monitor/process-log",apiMarketMonitorProcessLog) }

func apiMarketMonitorProcessLog(w http.ResponseWriter,r *http.Request){
 setCORS(w);w.Header().Set("Content-Type","application/json; charset=utf-8")
 if r.Method==http.MethodOptions {w.WriteHeader(http.StatusNoContent);return}
 if r.Method!=http.MethodGet {http.Error(w,`{"error":"Method not allowed"}`,http.StatusMethodNotAllowed);return}
 env,err:=readDBEnv("/home/rsapp/Application/Web_host/KSH/db.env")
 if err!=nil {writeProcessLogJSON(w,http.StatusServiceUnavailable,processLogResponse{Records:[]processLogRecord{},Errors:[]processLogError{{Server:"configuration",Error:"Cannot read database configuration file"}}});return}
 configs:=[]processDBConfig{
  {server:"74",host:env["DB74_HOST"],port:env["DB74_PORT"],name:env["DB74_NAME"],user:env["DB74_USER"],pass:env["DB74_PASSWORD"]},
  {server:"168",host:env["DB168_HOST"],port:env["DB168_PORT"],name:env["DB168_NAME"],user:env["DB168_USER"],pass:env["DB168_PASSWORD"]},
  {server:"166",host:env["DB166_HOST"],port:env["DB166_PORT"],name:env["DB166_NAME"],user:env["DB166_USER"],pass:env["DB166_PASSWORD"]},
 }
 result:=processLogResponse{Records:[]processLogRecord{},Errors:[]processLogError{}}
 for _,cfg:=range configs {
  if cfg.port=="" {cfg.port="3306"}
  if cfg.host==""||cfg.name==""||cfg.user==""||cfg.pass=="" {result.Errors=append(result.Errors,processLogError{Server:cfg.server,Error:"Database connection settings are incomplete in db.env"});continue}
  ctx,cancel:=context.WithTimeout(r.Context(),12*time.Second);records,qerr:=queryProcessLog(ctx,cfg);cancel()
  if qerr!=nil {result.Errors=append(result.Errors,processLogError{Server:cfg.server,Error:qerr.Error()});continue}
  result.Records=append(result.Records,records...)
 }
 writeProcessLogJSON(w,http.StatusOK,result)
}
func queryProcessLog(ctx context.Context,cfg processDBConfig)([]processLogRecord,error){
 dsn:=fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?timeout=5s&readTimeout=8s&writeTimeout=5s&parseTime=true",cfg.user,cfg.pass,cfg.host,cfg.port,cfg.name)
 db,err:=sql.Open("mysql",dsn);if err!=nil{return nil,fmt.Errorf("database setup failed")};defer db.Close()
 db.SetMaxOpenConns(1);db.SetMaxIdleConns(0)
 if err=db.PingContext(ctx);err!=nil{return nil,fmt.Errorf("database connection failed; check host, database, credentials and permissions")}
 const query=`SELECT PROCESS_DATE, EXCH_ID, SEGMENT, PROCESS, SUCCESS_FLAG, PROCESS_STATUS, SUCCESS, REJECTED, UPLOAD_FILE_NAME, FILE_NAME
 FROM RMS_PROCESS_LOG
 WHERE PROCESS_DATE >= CURDATE()
 AND PROCESS_DATE < CURDATE() + INTERVAL 1 DAY
 AND (SUCCESS_FLAG <> 'Y' OR SUCCESS_FLAG IS NULL)
 ORDER BY PROCESS_DATE DESC`
 rows,err:=db.QueryContext(ctx,query);if err!=nil{return nil,fmt.Errorf("RMS_PROCESS_LOG query failed; check table access and schema")};defer rows.Close()
 out:=[]processLogRecord{}
 for rows.Next(){
  var r processLogRecord
  var d,e,s,p,sf,ps,sc,rj,uf,fn sql.NullString
  if err:=rows.Scan(&d,&e,&s,&p,&sf,&ps,&sc,&rj,&uf,&fn);err!=nil{return nil,fmt.Errorf("could not read RMS_PROCESS_LOG result")}
  r.Server=cfg.server;r.ProcessDate=nv(d);r.ExchID=nv(e);r.Segment=nv(s);r.Process=nv(p);r.SuccessFlag=nv(sf);r.ProcessStatus=nv(ps);r.Success=nv(sc);r.Rejected=nv(rj);r.UploadFileName=nv(uf);r.FileName=nv(fn);out=append(out,r)
 }
 if err:=rows.Err();err!=nil{return nil,fmt.Errorf("error reading RMS_PROCESS_LOG rows")}
 return out,nil
}
func nv(v sql.NullString)interface{}{if !v.Valid{return nil};return v.String}
func readDBEnv(path string)(map[string]string,error){
 b,err:=os.ReadFile(path);if err!=nil{return nil,err};m:=map[string]string{}
 for _,line:=range strings.Split(string(b),"\n"){
  line=strings.TrimSpace(line);if line==""||strings.HasPrefix(line,"#"){continue};line=strings.TrimPrefix(line,"export ")
  k,v,ok:=strings.Cut(line,"=");if !ok{continue};k=strings.TrimSpace(k);v=strings.TrimSpace(v)
  if len(v)>=2&&((v[0]=='"'&&v[len(v)-1]=='"')||(v[0]=='\''&&v[len(v)-1]=='\'')){v=v[1:len(v)-1]}
  m[k]=v
 }
 return m,nil
}
func writeProcessLogJSON(w http.ResponseWriter,status int,v interface{}){w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
