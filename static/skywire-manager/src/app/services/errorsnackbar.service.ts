import { Injectable } from '@angular/core';
import {MatSnackBar} from "@angular/material";
import {MatSnackBarConfig} from "@angular/material/snack-bar/typings/snack-bar-config";
import {MatSnackBarRef} from "@angular/material/snack-bar/typings/snack-bar-ref";
import {SimpleSnackBar} from "@angular/material/snack-bar/typings/simple-snack-bar";

@Injectable({
  providedIn: 'root'
})
export class ErrorsnackbarService {

  constructor(private snackbar: MatSnackBar) { }

  open(message: string, action?: string, config: MatSnackBarConfig = {}): MatSnackBarRef<SimpleSnackBar>
  {
    config = {...config, panelClass: 'error-snack-bar'};
    return this.snackbar.open(message, action, config);
  }
}
