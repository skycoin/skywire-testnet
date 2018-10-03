import { Component, DoCheck, ElementRef, Input, IterableDiffers, OnInit, ViewChild } from '@angular/core';
import { Chart } from 'chart.js';

@Component({
  selector: 'app-line-chart',
  templateUrl: './line-chart.component.html',
  styleUrls: ['./line-chart.component.scss']
})
export class LineChartComponent implements OnInit, DoCheck {
  @ViewChild('chart') chartElement: ElementRef;
  @Input() data: number[];
  chart: any;

  private differ: any;

  constructor(
    private differs: IterableDiffers,
  ) {
    this.differ = differs.find([]).create(null);
  }

  ngOnInit() {
    this.chart = new Chart(this.chartElement.nativeElement, {
      type: 'line',
      data: {
        labels: Array.from(Array(this.data.length).keys()),
        datasets: [{
          data: this.data,
          backgroundColor: ['#0B6DB0'],
          borderColor: ['#0B6DB0'],
          borderWidth: 1,
        }],
      },
      options: {
        events: [],
        legend: { display: false },
        tooltips: { enabled: false },
        scales: {
          yAxes: [{ display: false }],
          xAxes: [{ display: false }],
        },
        elements: { point: { radius: 0 } },
      },
    });
  }

  ngDoCheck() {
    const changes = this.differ.diff(this.data);

    if (changes && this.chart) {
      this.chart.update();
    }
  }
}
