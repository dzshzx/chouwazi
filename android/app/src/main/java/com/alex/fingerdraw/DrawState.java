package com.alex.fingerdraw;
import java.util.*;
import java.security.SecureRandom;
/** Platform-independent round state. Pointer IDs, not array indices, identify players. */
public final class DrawState {
 public static final int MAX=10;
 public static final long WAIT=3800;
 public final LinkedHashMap<Integer,Point> points=new LinkedHashMap<>();
 public long deadline=0;
 public boolean done=false;
 public int winner=-1;
 private final Random random;
 public DrawState(){this(new SecureRandom());}
 public DrawState(Random random){this.random=random;}
 public static final class Point {
  public float x,y; public final int slot;
  Point(float x,float y,int slot){this.x=x;this.y=y;this.slot=slot;}
 }
 public void add(int id,float x,float y,long now){
  if(done || points.containsKey(id) || points.size()>=MAX)return;
  boolean[] used=new boolean[MAX]; for(Point p:points.values())used[p.slot]=true;
  int slot=0;while(used[slot])slot++;
  points.put(id,new Point(x,y,slot)); restart(now);
 }
 public void move(int id,float x,float y){if(!done){Point p=points.get(id);if(p!=null){p.x=x;p.y=y;}}}
 public void remove(int id,long now){if(!done && points.remove(id)!=null)restart(now);}
 private void restart(long now){deadline=points.size()>=2?now+WAIT:0;}
 public boolean tick(long now){
  if(done || deadline==0 || now<deadline)return false;
  winner=new ArrayList<>(points.keySet()).get(random.nextInt(points.size()));done=true;deadline=0;return true;
 }
 public void reset(){points.clear();deadline=0;done=false;winner=-1;}
}
