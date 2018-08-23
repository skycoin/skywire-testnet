import {Component, HostListener, Input, OnInit, ViewChild} from '@angular/core';
import {MatMenuTrigger, MatSnackBar} from '@angular/material';
import { TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-copy-to-clipboard-text',
  templateUrl: './copy-to-clipboard-text.component.html',
  styleUrls: ['./copy-to-clipboard-text.component.css'],
  host: {'class': 'copy-to-clipboard-container'}
})
export class CopyToClipboardTextComponent implements OnInit {
  @ViewChild(MatMenuTrigger) trigger: MatMenuTrigger;
  @Input() text: string;
  @Input() shortTextLength = 6;
  @Input() short = false;
  tooltipText: string;
  fullText: string;

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
      this.shortenText();
    } else {
      this.tooltipText = 'copy.click-to-copy';
    }
  }

  private shortenText() {
    const lastTextIndex = this.text.length,
        prefix = this.text.slice(0, 6),
        sufix = this.text.slice((lastTextIndex - this.shortTextLength), lastTextIndex);

    this.text = `${prefix}...${sufix}`;
  }

  @HostListener('click') onClick() {
    this.trigger.openMenu();
  }

  public onCopyToClipboardClicked() {
    this.translate.get('copy.copied').subscribe(str => {
      this.openSnackBar(str);
    });
  }
}
