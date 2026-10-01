import com.alex.fingerdraw.DrawState;
import java.util.Random;
public class DrawStateTest {
 static void check(boolean b){if(!b)throw new AssertionError();}
 public static void main(String[] args){
  DrawState s=new DrawState(new Random(42));s.add(17,1,1,100);check(s.deadline==0);s.add(29,2,2,200);check(s.deadline==4000);check(!s.tick(3999));s.remove(17,3999);check(s.deadline==0);s.add(44,1,1,5000);check(s.deadline==8800);s.move(44,8,9);check(s.deadline==8800);check(s.tick(8800));check(s.points.containsKey(s.winner));int win=s.winner;s.remove(win,9000);check(s.winner==win&&s.points.size()==2);s.reset();check(!s.done&&s.points.isEmpty());
  for(int i=0;i<11;i++)s.add(i,0,0,0);check(s.points.size()==10);s.remove(4,100);s.add(14,0,0,200);check(s.points.get(14).slot==4);check(s.tick(4000));
  int[] counts=new int[10];for(int round=0;round<100000;round++){s.reset();for(int i=0;i<10;i++)s.add(i,0,0,0);s.tick(3800);counts[s.winner]++;}for(int count:counts)check(count>9500&&count<10500);
  System.out.println("PASS: pointer IDs, 10-touch cap, restart, removal, result freeze, reset, distribution smoke test");
 }
}
