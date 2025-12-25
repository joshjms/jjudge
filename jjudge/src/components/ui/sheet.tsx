import * as React from "react";
import * as SheetPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";

import { cn } from "@/lib/utils";

const Sheet = SheetPrimitive.Root;
const SheetTrigger = SheetPrimitive.Trigger;
const SheetClose = SheetPrimitive.Close;

const SheetPortal = ({
	...props
}: SheetPrimitive.DialogPortalProps) => (
	<SheetPrimitive.Portal {...props} />
);
SheetPortal.displayName = SheetPrimitive.Portal.displayName;

const SheetOverlay = React.forwardRef<
	React.ElementRef<typeof SheetPrimitive.Overlay>,
	React.ComponentPropsWithoutRef<typeof SheetPrimitive.Overlay>
>(({ className, ...props }, ref) => (
	<SheetPrimitive.Overlay
		className={cn(
			"fixed inset-0 z-50 bg-background/60 backdrop-blur-sm transition-opacity data-[state=closed]:opacity-0 data-[state=open]:opacity-100",
			className
		)}
		{...props}
		ref={ref}
	/>
));
SheetOverlay.displayName = SheetPrimitive.Overlay.displayName;

const SheetContent = React.forwardRef<
	React.ElementRef<typeof SheetPrimitive.Content>,
	React.ComponentPropsWithoutRef<typeof SheetPrimitive.Content>
>(({ className, children, ...props }, ref) => (
	<SheetPortal>
		<SheetOverlay />
		<SheetPrimitive.Content
			ref={ref}
			className={cn(
				"fixed inset-y-0 right-0 z-50 flex w-3/4 max-w-sm flex-col rounded-l-3xl border border-border bg-background/95 p-6 text-foreground shadow-xl backdrop-blur transition-transform data-[state=closed]:translate-x-full data-[state=open]:translate-x-0",
				className
			)}
			{...props}
		>
			{children}
			<SheetPrimitive.Close className="absolute right-3 top-3 rounded-full p-1 text-muted-foreground transition hover:text-foreground">
				<X className="h-5 w-5" />
				<span className="sr-only">Close</span>
			</SheetPrimitive.Close>
		</SheetPrimitive.Content>
	</SheetPortal>
));
SheetContent.displayName = SheetPrimitive.Content.displayName;

const SheetHeader = ({
	className,
	...props
}: React.HTMLAttributes<HTMLDivElement>) => (
	<div className={cn("space-y-1.5 text-center sm:text-left", className)} {...props} />
);
SheetHeader.displayName = "SheetHeader";

const SheetFooter = ({
	className,
	...props
}: React.HTMLAttributes<HTMLDivElement>) => (
	<div
		className={cn("flex flex-col-reverse gap-2 sm:flex-row sm:justify-end", className)}
		{...props}
	/>
);
SheetFooter.displayName = "SheetFooter";

const SheetTitle = React.forwardRef<
	React.ElementRef<typeof SheetPrimitive.Title>,
	React.ComponentPropsWithoutRef<typeof SheetPrimitive.Title>
>(({ className, ...props }, ref) => (
	<SheetPrimitive.Title
		ref={ref}
		className={cn("text-lg font-semibold text-foreground", className)}
		{...props}
	/>
));
SheetTitle.displayName = SheetPrimitive.Title.displayName;

const SheetDescription = React.forwardRef<
	React.ElementRef<typeof SheetPrimitive.Description>,
	React.ComponentPropsWithoutRef<typeof SheetPrimitive.Description>
>(({ className, ...props }, ref) => (
	<SheetPrimitive.Description
		ref={ref}
		className={cn("text-sm text-muted-foreground", className)}
		{...props}
	/>
));
SheetDescription.displayName = SheetPrimitive.Description.displayName;

export {
	Sheet,
	SheetTrigger,
	SheetClose,
	SheetContent,
	SheetHeader,
	SheetFooter,
	SheetTitle,
	SheetDescription,
};
