function r(t){const e=Date.now()-t;if(e<1e3)return"刚刚";if(e<6e4)return`${Math.round(e/1e3)} 秒前`;if(e<36e5)return`${Math.round(e/6e4)} 分前`;if(e<864e5)return`${Math.round(e/36e5)} 小时前`;const n=new Date(t);return`${n.getMonth()+1}/${n.getDate()}`}export{r as f};
//# sourceMappingURL=time-CSIxpxP1.js.map
