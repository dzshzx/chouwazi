package com.alex.fingerdraw;
import android.app.Activity;
import android.os.*;
import android.content.*;
import android.graphics.*;
import android.view.*;
import java.util.*;

public class MainActivity extends Activity {
 private GameView game;
 @Override public void onCreate(Bundle b){super.onCreate(b);getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);getWindow().setStatusBarColor(Color.rgb(13,17,27));getWindow().setNavigationBarColor(Color.rgb(13,17,27));game=new GameView(this);android.widget.FrameLayout root=new android.widget.FrameLayout(this);root.setBackgroundColor(0xFF0D111B);root.addView(game);root.setOnApplyWindowInsetsListener(new View.OnApplyWindowInsetsListener(){public WindowInsets onApplyWindowInsets(View v,WindowInsets insets){v.setPadding(insets.getSystemWindowInsetLeft(),insets.getSystemWindowInsetTop(),insets.getSystemWindowInsetRight(),insets.getSystemWindowInsetBottom());return insets.consumeSystemWindowInsets();}});setContentView(root);root.requestApplyInsets();}
 @Override protected void onPause(){super.onPause();game.pause();}
 @Override protected void onResume(){super.onResume();if(game!=null)game.resume();}
 final class GameView extends View {
  final DrawState state=new DrawState(); final Paint p=new Paint(3);
  final int[] colors={0xFFB5F36D,0xFF8CB8FF,0xFFFFADCA,0xFFFFD478,0xFFC3A0FF,0xFF69DDCA,0xFFFF967F,0xFF98E2FF,0xFFE7A7EC,0xFFE5EA8B};
  final RectF retry=new RectF(),sound=new RectF();
  final Set<Integer> physical=new HashSet<>();
  float d; boolean active=true, vibration=true, resetArmed=false; int buttonPointer=-1; int lastSecond=-1;
  GameView(Context c){super(c);d=getResources().getDisplayMetrics().density;setFocusable(true);setContentDescription("抽袜子。至少两根手指同时按住，倒计时后随机选中一根，最多十根。");}
  void pause(){active=false;physical.clear();buttonPointer=-1;resetArmed=false;if(!state.done)state.reset();}
  void resume(){active=true;invalidate();}
  void buzz(long ms){if(!vibration)return;Vibrator v=(Vibrator)getSystemService(VIBRATOR_SERVICE);if(v==null||!v.hasVibrator())return;if(Build.VERSION.SDK_INT>=26)v.vibrate(VibrationEffect.createOneShot(ms,VibrationEffect.DEFAULT_AMPLITUDE));else v.vibrate(ms);}
  void text(Canvas c,String s,float x,float y,float size,int color,boolean bold){p.setStyle(Paint.Style.FILL);p.setColor(color);p.setTextSize(size*d);p.setTypeface(bold?Typeface.create("sans-serif",Typeface.BOLD):Typeface.create("sans-serif",Typeface.NORMAL));p.setTextAlign(Paint.Align.CENTER);c.drawText(s,x,y,p);}
  void ring(Canvas c,float x,float y,float r,int color,float width){p.setStyle(Paint.Style.STROKE);p.setStrokeWidth(width*d);p.setColor(color);c.drawCircle(x,y,r,p);p.setStyle(Paint.Style.FILL);}
  @Override protected void onDraw(Canvas c){
   long now=SystemClock.uptimeMillis(); if(active && state.tick(now)){buzz(180);announceForAccessibility("已抽中第"+(state.points.get(state.winner).slot+1)+"号");}
   float w=getWidth(),h=getHeight(),cx=w/2;
   c.drawColor(0xFF0D111B);
   p.setColor(0xFF18202C);for(float x=20*d;x<w;x+=28*d)for(float y=160*d;y<h-120*d;y+=28*d)c.drawCircle(x,y,0.8f*d,p);
   text(c,"CHOUWAZI",cx,35*d,10,0xFF8B96AA,true);
   text(c,getContext().getString(R.string.app_name),cx,73*d,28,Color.WHITE,true);
   text(c,"把选择，交给这一刻",cx,100*d,12,0xFF8B96AA,false);
   sound.set(w-88*d,8*d,w-10*d,43*d);
   text(c,vibration?"震动 开":"震动 关",w-49*d,30*d,11,0xFF8B96AA,false);
   String status=state.done?"第 "+(state.points.get(state.winner).slot+1)+" 号 · 就是你！":state.points.size()+" / 10 根手指已加入";
   text(c,status,cx,140*d,15,state.done?colors[state.points.get(state.winner).slot]:0xFFB5F36D,true);
   float mid=Math.max(215*d,h*0.47f);
   if(!state.done && state.points.size()<2){
    ring(c,cx,mid,48*d,0xFF283C38,2);ring(c,cx,mid,35*d,0xFF536D49,2);
    text(c,"+",cx,mid+12*d,36,0xFFB5F36D,false);
    text(c,state.points.isEmpty()?"大家一起按住屏幕":"再来一根手指",cx,mid+87*d,20,Color.WHITE,true);
    text(c,"至少 2 根 · 最多 10 根",cx,mid+113*d,13,0xFF8B96AA,false);
   }
   if(!state.done && state.deadline>0){
    long remaining=state.deadline-now;
    int sec=(int)Math.ceil(Math.min(3000,remaining)/1000.0);
    if(remaining<=3000 && sec!=lastSecond){lastSecond=sec;buzz(22);}
    text(c,remaining>3000?"准备":String.valueOf(Math.max(1,sec)),cx,mid+20*d,remaining>3000?40:76,0xFFE7EDE5,true);
    text(c,"保持按住，即将随机选出一位",cx,mid+59*d,13,0xFF8B96AA,false);
   }
   for(Map.Entry<Integer,DrawState.Point> e:state.points.entrySet()){
    DrawState.Point f=e.getValue();int col=colors[f.slot];boolean selected=state.done&&e.getKey()==state.winner;
    float r=(selected?49:36)*d+(float)Math.sin(now/230.0+f.slot)*2*d;
    p.setColor(col);p.setAlpha(state.done&&!selected?12:35);c.drawCircle(f.x,f.y,r+12*d,p);p.setAlpha(255);
    ring(c,f.x,f.y,r,state.done&&!selected?0xFF384050:col,selected?5:3);
    p.setColor(state.done&&!selected?0xFF202735:col);c.drawCircle(f.x,f.y,23*d,p);
    text(c,""+(f.slot+1),f.x,f.y+6*d,17,0xFF0D111B,true);
    if(selected){text(c,"就是你！",f.x,Math.max(170*d,f.y-r-18*d),19,col,true);
     for(int i=0;i<12;i++){double a=i*Math.PI/6;float rr=r+22*d;p.setColor(col);c.drawCircle(f.x+(float)Math.cos(a)*rr,f.y+(float)Math.sin(a)*rr,2*d,p);}}
   }
   retry.set(28*d,h-93*d,w-28*d,h-39*d);
   if(state.done){p.setColor(physical.isEmpty()?0xFFB5F36D:0xFF344336);c.drawRoundRect(retry,18*d,18*d,p);text(c,physical.isEmpty()?"再来一轮":"松开所有手指后再来一轮",cx,h-59*d,16,physical.isEmpty()?0xFF152010:0xFFA1B394,true);}
   else{text(c,"新增或移开手指，会重新倒计时",cx,h-66*d,12,0xFF8B96AA,false);}
   text(c,"离线随机抽选 · 实际触点数取决于手机",cx,h-18*d,10,0xFF687488,false);
   if(active && (state.deadline>0 || !state.points.isEmpty()))postInvalidateOnAnimation();
  }
  @Override public boolean onTouchEvent(android.view.MotionEvent e){
   int action=e.getActionMasked(),index=e.getActionIndex(),id=e.getPointerId(index);long now=SystemClock.uptimeMillis();
   if(action==MotionEvent.ACTION_CANCEL){physical.clear();buttonPointer=-1;resetArmed=false;if(!state.done)state.reset();invalidate();return true;}
   if(action==MotionEvent.ACTION_DOWN || action==MotionEvent.ACTION_POINTER_DOWN){
    boolean empty=physical.isEmpty();physical.add(id);
    if(action==MotionEvent.ACTION_DOWN && empty && sound.contains(e.getX(index),e.getY(index))){vibration=!vibration;buttonPointer=id;invalidate();return true;}
    if(state.done){if(empty && retry.contains(e.getX(index),e.getY(index))){resetArmed=true;buttonPointer=id;}else resetArmed=false;}
    else if(buttonPointer<0){state.add(id,e.getX(index),e.getY(index),now);lastSecond=-1;buzz(15);}
   }
   if(!state.done && buttonPointer<0)for(int i=0;i<e.getPointerCount();i++)state.move(e.getPointerId(i),e.getX(i),e.getY(i));
   if(action==MotionEvent.ACTION_UP || action==MotionEvent.ACTION_POINTER_UP){
    physical.remove(id);
    if(id==buttonPointer){if(resetArmed && physical.isEmpty() && retry.contains(e.getX(index),e.getY(index))){state.reset();lastSecond=-1;performClick();}buttonPointer=-1;resetArmed=false;}
    else {state.remove(id,now);lastSecond=-1;}
   }
   invalidate();return true;
  }
  @Override public boolean performClick(){super.performClick();return true;}
 }
}
