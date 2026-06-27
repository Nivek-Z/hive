package zhangkaiwen;

import java.time.LocalDate;

/** 按日计数行（个人热力图 / 蜂巢活跃统计共用） */
public class HeatRow {

    private LocalDate date;
    private Integer count;

    public LocalDate getDate() { return date; }
    public void setDate(LocalDate date) { this.date = date; }
    public Integer getCount() { return count; }
    public void setCount(Integer count) { this.count = count; }
}
