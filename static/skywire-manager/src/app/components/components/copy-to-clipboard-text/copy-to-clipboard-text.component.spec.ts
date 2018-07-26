import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { CopyToClipboardTextComponent } from './copy-to-clipboard-text.component';

describe('CopyToClipboardTextComponent', () => {
  let component: CopyToClipboardTextComponent;
  let fixture: ComponentFixture<CopyToClipboardTextComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ CopyToClipboardTextComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(CopyToClipboardTextComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
