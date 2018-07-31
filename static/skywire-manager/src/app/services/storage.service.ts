import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class StorageService implements Storage
{
  private storage: Storage;

  constructor(private namespace: string)
  {
    this.storage = localStorage;
  }

  static getNamedStorage(namespace: string): Storage
  {
    return new StorageService(namespace);
  }

  [key: string]: any;

  readonly length: number;

  private namespacedKey(key: string)
  {
    return `${this.namespace}-${key}`;
  }

  clear(): void
  {
    return this.storage.clear();
  }

  getItem(key: string): string | null {
    return this.storage.getItem(this.namespacedKey(key));
  }

  key(index: number): string | null {
    return this.storage.key(index);
  }

  removeItem(key: string): void {
    this.storage.removeItem(this.namespacedKey(key))
  }

  setItem(key: string, value: string): void {
    this.storage.setItem(this.namespacedKey(key), value);
  }
}
