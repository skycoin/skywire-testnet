import { Injectable } from '@angular/core';
import { MdDialog } from '@angular/material';
import { AlertDialogComponent } from '../../components';

@Injectable()
export class ToolService {
  constructor(private dialog: MdDialog) { }
  alert(title: string, message: string = '', type: string = 'info') {
    const ref = this.dialog.open(
      AlertDialogComponent,
      {
        position: { top: '10%' },
        panelClass: 'alert-dialog-panel',
        backdropClass: 'alert-backdrop',
        width: '23rem'
      });
    ref.componentInstance.title = title;
    ref.componentInstance.message = message;
    ref.componentInstance.type = type;
    return ref.afterClosed();
  }
  ShowInfoAlert(title: string = 'Information', message: string = '') {
    return this.alert(title, message, 'info');
  }
  ShowSuccessAlert(title: string = 'Information', message: string = '') {
    return this.alert(title, message, 'success');
  }
  ShowWarningAlert(title: string = 'Information', message: string = '') {
    return this.alert(title, message, 'warning');
  }
  ShowDangerAlert(title: string = 'Information', message: string = '') {
    return this.alert(title, message, 'danger');
  }
  padZero(str: string, len?: number) {
    len = len || 2;
    const zeros = new Array(len).join('0');
    return (zeros + str).slice(-len);
  }

  getRandomColor() {
    const color = Math.round(Math.random() * 0x1000000).toString(16);
    return '#' + this.padZero(color, 6);
  }
  invertColor(hex: string, bw: boolean = true) {
    if (hex.indexOf('#') === 0) {
      hex = hex.slice(1);
    }
    // convert 3-digit hex to 6-digits.
    if (hex.length === 3) {
      hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2];
    }
    if (hex.length !== 6) {
      throw new Error('Invalid HEX color.');
    }
    const r = parseInt(hex.slice(0, 2), 16),
      g = parseInt(hex.slice(2, 4), 16),
      b = parseInt(hex.slice(4, 6), 16);
    if (bw) {
      return (r * 0.299 + g * 0.587 + b * 0.114) > 186
        ? '#000000'
        : '#FFFFFF';
    }
    return '#' + this.padZero((255 - r).toString(16)) + this.padZero((255 - g).toString(16)) + this.padZero((255 - b).toString(16));
  }
  getRandomMatch() {
    const bg = this.getRandomColor();
    return { bg: bg, color: this.invertColor(bg) };
  }
}
