import { Injectable } from '@angular/core';
import swal from 'sweetalert2';

@Injectable()
export class AlertService {
  task = null;
  constructor() { }
  success(message: string) {
    return swal('Success', message, 'success');
  }
  error(message: string) {
    return swal('Error', message, 'error');
  }
  warning(message: string) {
    return swal('Warning', message, 'warning');
  }
  confirm(title: string, message: string, type: AlertType) {
    return swal({
      title: title,
      text: message,
      type: type,
      showCancelButton: true,
      confirmButtonColor: '#3085d6',
      cancelButtonColor: '#d33',
      confirmButtonText: 'Yes'
    });
  }
  timer(message: string, timer = 3000) {
    return swal({
      title: 'Please wait',
      text: message,
      timer: timer,
      allowOutsideClick: false,
      allowEscapeKey: false,
      onOpen: () => {
        swal.showLoading();
      }
    });
  }
  // repeat(message: string, timer = 3000, repeatFn: Function, count = 3, delay = 1000) {
  //   this.timer(message, timer).then((r1) => {
  //     if (r1.dismiss === 'dismiss') {
  //       this.timer('repeat...', timer * count - 1).then((r2) => {
  //         if (r2.dismiss === 'dismiss') {
  //           this.task = setInterval(() => {
  //             repeatFn();
  //           }, delay);
  //         }
  //       });
  //     }
  //   });
  // }
  close() {
    clearInterval(this.task);
    swal.close();
  }
}
export type AlertType = 'success' | 'error' | 'warning' | 'info' | 'question' | undefined;
