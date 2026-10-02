(function(){
  if(location.pathname.endsWith("/login.html")) return;
  if(sessionStorage.getItem("rs_authenticated") !== "1"){
    const next=location.pathname.split("/").pop()||"index.html";
    location.replace("login.html?next="+encodeURIComponent(next));
  }
})();