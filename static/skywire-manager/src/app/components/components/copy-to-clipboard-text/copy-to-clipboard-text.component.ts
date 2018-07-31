import {Component, HostListener, Input, OnDestroy, OnInit, ViewChild} from '@angular/core';
import {MatMenuTrigger, MatSnackBar, MatTooltip} from "@angular/material";

@Component({
  selector: 'app-copy-to-clipboard-text',
  templateUrl: './copy-to-clipboard-text.component.html',
  styleUrls: ['./copy-to-clipboard-text.component.css']
})
export class CopyToClipboardTextComponent implements OnInit, OnDestroy
{
  @ViewChild(MatMenuTrigger) trigger: MatMenuTrigger;
  @Input() text: string;
  @Input() shortTextLength: number = 6;
  @Input() short: boolean = false;
  tooltipText: string;
  fullText: string;
  @ViewChild('copyTooltip') tooltip: MatTooltip;

  constructor(public snackBar: MatSnackBar) {}

  openSnackBar(message: string)
  {
    this.snackBar.open(message, null, {
      duration: 2000,
    });
  }

  ngOnInit()
  {
    this.fullText = this.text;
    if (this.short)
    {
      this.tooltipText = "Click to see full text";
      this.shortenText();
    }
    else
    {
      this.tooltipText = "Click to copy";
    }
  }

  ngOnDestroy()
  {

  }

  private shortenText()
  {
    let lastTextIndex = this.text.length,
        prefix = this.text.slice(0, 6),
        sufix = this.text.slice((lastTextIndex - this.shortTextLength), lastTextIndex);

    this.text = `${prefix}...${sufix}`;
  }

  @HostListener('click') onClick()
  {
    this.trigger.openMenu();
  }

  public onCopyToClipboardClicked()
  {
    this.openSnackBar('Copied!')
  }
}
