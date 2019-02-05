import {Component, HostListener, Input, OnInit, ViewChild} from '@angular/core';
import {MatMenuTrigger, MatSnackBar} from '@angular/material';
import { TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-copy-to-clipboard-text',
  templateUrl: './copy-to-clipboard-text.component.html',
  styleUrls: ['./copy-to-clipboard-text.component.css']
})
export class CopyToClipboardTextComponent implements OnInit {
  @Input() public short = false;
  @Input() text: string;
  @Input() shortTextLength = 6;
  // @ViewChild(MatMenuTrigger) trigger: MatMenuTrigger;
  tooltipText: string;
  fullText: string;

  get shortText() {
    const lastTextIndex = this.text.length,
      prefix = this.text.slice(0, this.shortTextLength),
      sufix = this.text.slice((lastTextIndex - this.shortTextLength), lastTextIndex);

    return `${prefix}...${sufix}`;
  }

  constructor(
    public snackBar: MatSnackBar,
    private translate: TranslateService,
  ) {}

  openSnackBar(message: string) {
    this.snackBar.open(message, null, {
      duration: 2000,
    });
  }

  ngOnInit() {
    this.fullText = this.text;
    if (this.short) {
      this.tooltipText = 'copy.click-to-see';
    } else {
      this.tooltipText = 'copy.click-to-copy';
    }
  }

  // @HostListener('click') onClick() {
  //   this.trigger.openMenu();
  // }

  public onCopyToClipboardClicked() {
    this.translate.get('copy.copied').subscribe(str => {
      this.openSnackBar(str);
    });
  }
}
