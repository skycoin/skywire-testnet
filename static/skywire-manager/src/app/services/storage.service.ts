import { Injectable } from '@angular/core';

const KEY_REFRESH_SECONDS: string = 'KEY_REFRESH_SECONDS';
const KEY_DEFAULT_LANG:    string = 'KEY_DEFAULT_LANG';

@Injectable({
  providedIn: 'root'
})
export class StorageService
{
  private storage: Storage;

  constructor()
  {
    this.storage = localStorage;
  }

  private static nodeLabelNamespace(nodeKey: string): string
  {
    return `${nodeKey}-label`;
  }

  public setNodeLabel(nodeKey: string, nodeLabel: string): void
  {
    this.storage.setItem(StorageService.nodeLabelNamespace(nodeKey), nodeLabel);
  }

  public getNodeLabel(nodeKey: string): string
  {
    return this.storage.getItem(StorageService.nodeLabelNamespace(nodeKey));
  }

  setRefreshTime(seconds: number)
  {
    this.storage.setItem(KEY_REFRESH_SECONDS, seconds.toString());
  }

  getRefreshTime(): number
  {
    return parseInt(this.storage.getItem(KEY_REFRESH_SECONDS));
  }

  setDefaultLanguage(lang: string): void
  {
    this.storage.setItem(KEY_DEFAULT_LANG, lang);
  }

  getDefaultLanguage(): string
  {
    return this.storage.getItem(KEY_DEFAULT_LANG) || "en";
  }
}
