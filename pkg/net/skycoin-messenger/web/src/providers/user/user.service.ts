import { Injectable } from '@angular/core';
import { HeadColorMatch } from '../socket/msg'
@Injectable()
export class UserService {
  randomMatch: Array<HeadColorMatch> = [
    { bg: '#fff', color: '#000' },
    { bg: '#d05454', color: '#fff' },
    { bg: '#6dd067', color: '#fff' },
    { bg: '#676fd0', color: '#fff' },
    { bg: '#e47ae1', color: '#fff' },
    { bg: '#67c1d0', color: '#fff' },
    { bg: '#000', color: '#fff' },
    { bg: '#ffef2d', color: '#000' },
    { bg: '#eaae27', color: '#fff' },
    { bg: '#fbd1dc', color: '#000' },
  ]
  constructor() { }
  padZero(str: string, len?: number) {
    len = len || 2;
    const zeros = new Array(len).join('0');
    return (zeros + str).slice(-len);
  }
  getRandomRgb() {
    const num = Math.round(0xffffff * Math.random());
    const r = num >> 16;
    const g = num >> 8 & 255;
    const b = num & 255;
    return [r, g, b];
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
